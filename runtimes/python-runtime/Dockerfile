FROM python:3.11-slim

WORKDIR /runtime

RUN pip install fastapi uvicorn

COPY runtime.py .

EXPOSE 8080

HEALTHCHECK --interval=2s --timeout=2s --start-period=5s --retries=5 \
    CMD python -c "import urllib.request, sys; \
    sys.exit(0) if urllib.request.urlopen('http://localhost:8080/health').getcode()==200 else sys.exit(1)"

CMD ["python", "runtime.py"]
