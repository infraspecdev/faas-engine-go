const http = require("http");
const path = require("path");

const PORT = 8080;

// ENV Variables
const FUNCTION_FILE = process.env.FUNCTION_FILE || "index.js";
const FUNCTION_EXPORT = process.env.FUNCTION_EXPORT || null;

let userModule;
try {
  userModule = require(path.join("/function", FUNCTION_FILE));
} catch (err) {
  console.error("Failed to load user function:", err);
  process.exit(1);
}


const handler = FUNCTION_EXPORT
  ? userModule[FUNCTION_EXPORT]
  : userModule;

if (typeof handler !== "function") {
  console.error("Exported handler is not a function");
  process.exit(1);
}

const server = http.createServer(async (req, res) => {

  if (req.method === "GET" && req.url === "/health") {
  res.writeHead(200);
  return res.end("ok");
}

  if (req.method !== "POST") {
    res.writeHead(405, { "Content-Type": "application/json" });
    return res.end(JSON.stringify({ error: "Method Not Allowed" }));
  }

  let body = "";

  req.on("data", chunk => {
    body += chunk;
  });

  req.on("end", async () => {
    try {
      const event = body ? JSON.parse(body) : {};

      const result = await handler(event);

      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ result }));
    } catch (err) {
      console.error("Function execution error:", err);

      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: err.message }));
    }
  });
});

server.listen(PORT, "0.0.0.0", () => {
  console.log(`Node FaaS runtime listening on port ${PORT}`);
});
