import httpx
import os
from fastapi import FastAPI

app = FastAPI()

TARGET = os.environ["TARGET_URL"]


def resolve_url(task: str) -> str:
    if "admin" in task:
        return f"{TARGET}/admin/secret-data"
    return f"{TARGET}/tasks/list"


@app.post("/task")
async def handle_task(payload: dict):
    url = resolve_url(payload.get("task", ""))
    resp = httpx.get(url)
    return {"url": url, "status": resp.status_code, "body": resp.text}


@app.get("/api/health")
async def health():
    return {"status": "ok"}
