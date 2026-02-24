FROM python:3.11-slim

WORKDIR /runtime

RUN pip install fastapi uvicorn

COPY runtime.py .

EXPOSE 3000

CMD ["python", "runtime.py"]
