FROM node:20-alpine

WORKDIR /runtime

COPY runtime.js .

EXPOSE 8080

HEALTHCHECK --interval=1s --timeout=1s --retries=5 \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["node", "runtime.js"]
