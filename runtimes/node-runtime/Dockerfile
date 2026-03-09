FROM node:20-alpine

WORKDIR /runtime

COPY runtime.js .

EXPOSE 8080

HEALTHCHECK --interval=1s --timeout=1s --retries=5 \
  CMD node -e "require('http').get('http://localhost:8080/health', r => process.exit(r.statusCode===200?0:1)).on('error',()=>process.exit(1))"

CMD ["node", "runtime.js"]
