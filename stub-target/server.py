from http.server import HTTPServer, BaseHTTPRequestHandler
import json


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps(
            {"ok": True, "path": self.path, "headers": dict(self.headers)}
        ).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        self.do_GET()

    def log_message(self, fmt, *args):
        print(fmt % args)


if __name__ == "__main__":
    HTTPServer(("0.0.0.0", 80), Handler).serve_forever()
