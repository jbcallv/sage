FROM python:3.12-slim
WORKDIR /app
RUN pip install httpx
COPY agent.py .
CMD ["python3", "agent.py"]
