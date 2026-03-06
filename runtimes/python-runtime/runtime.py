import importlib.util
import os
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
import uvicorn

FUNCTION_FILE = os.getenv("FUNCTION_FILE", "index.py")
FUNCTION_PATH = f"/function/{FUNCTION_FILE}"

# Dynamically load user function
spec = importlib.util.spec_from_file_location("user_module", FUNCTION_PATH)
user_module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(user_module)

if not hasattr(user_module, "handler"):
    raise Exception("User function must define 'handler(event)'")

handler = user_module.handler

app = FastAPI()

@app.post("/")
async def invoke(request: Request):
    try:
        event = await request.json()
        result = handler(event)
        return {"result": result}
    except Exception as e:
        return JSONResponse(status_code=500, content={"error": str(e)})

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=3000)
