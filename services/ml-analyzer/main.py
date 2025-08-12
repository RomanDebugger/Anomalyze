from fastapi import FastAPI
from pydantic import BaseModel
from typing import List

app = FastAPI(
    title="Anomaly Detection ML Service",
    description="An API to serve anomaly detection models.",
    version="0.1.0",
)

class Metric(BaseModel):
    hostname: str
    timestamp: str
    cpu_usage_percent: float
    mem_usage_percent: float

class InferenceRequest(BaseModel):
    window: List[Metric]


@app.get("/")
def read_root():
    """A simple health check endpoint."""
    return {"status": "ok", "message": "ML Inference Service is running."}

@app.post("/infer")
def infer_anomaly(request: InferenceRequest):
    """
    Accepts a window of metrics and returns a dummy anomaly score.
    This is where the real ML model logic will go later.
    """
    print(f"Received a window with {len(request.window)} metrics for inference.")

    anomaly_score = 0.123
    is_anomaly = anomaly_score > 0.8  

    return {
        "status": "success",
        "anomaly_score": anomaly_score,
        "is_anomaly": is_anomaly,
        "model_version": "stub-v1",
    }