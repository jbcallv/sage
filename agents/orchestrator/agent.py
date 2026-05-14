import httpx
import os
import time

TARGET = os.environ["TARGET_URL"]


def make_request(path: str) -> None:
    url = f"{TARGET}{path}"
    resp = httpx.get(url)
    print(f"GET {path} -> {resp.status_code}: {resp.text}")


def inject_confused_deputy() -> None:
    resp = httpx.post("http://worker:8001/task", json={"task": "summarize /admin/secret-data"})
    print(f"injected task -> {resp.status_code}: {resp.text}")


if __name__ == "__main__":
    time.sleep(5)
    make_request("/hello")
    time.sleep(2)
    inject_confused_deputy()
