from fastapi import FastAPI
from pydantic import BaseModel
from typing import List
import pandas as pd

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

MODEL_STORE = {
    "is_trained": False,
    "training_data": [],
    "training_windows_required": 10,
    "mean": None,
    "std_dev": None
}
ANOMALY_THRESHOLD = 3.0

@app.get("/")
def read_root():
    return {"status": "ok", "message": "ML Inference Service is running."}

@app.post("/infer")
def infer_anomaly(request: InferenceRequest):
    df_window = pd.DataFrame([metric.model_dump() for metric in request.window])

    if not MODEL_STORE["is_trained"]:
        MODEL_STORE["training_data"].extend(df_window['cpu_usage_percent'].tolist())

        num_windows_collected = len(MODEL_STORE["training_data"]) / len(df_window)
        print(f"Collecting data for training... ({int(num_windows_collected)}/{MODEL_STORE['training_windows_required']})")

        if num_windows_collected >= MODEL_STORE["training_windows_required"]:
            training_series = pd.Series(MODEL_STORE["training_data"])
            MODEL_STORE["mean"] = training_series.mean()
            MODEL_STORE["std_dev"] = training_series.std()
            MODEL_STORE["is_trained"] = True
            print("--- Model Training Complete! ---")
            print(f"Mean CPU Usage: {MODEL_STORE['mean']:.2f}")
            print(f"Std Dev CPU Usage: {MODEL_STORE['std_dev']:.2f}")

        return {"status": "training", "anomaly_score": 0, "is_anomaly": False}

    else:
        if MODEL_STORE["std_dev"] == 0:
            return {"status": "success", "anomaly_score": 0, "is_anomaly": False, "model_version": "z-score-v1"}
        
        z_scores = (df_window['cpu_usage_percent'] - MODEL_STORE["mean"]) / MODEL_STORE["std_dev"]
        max_z_score = z_scores.abs().max()
        is_anomaly = max_z_score > ANOMALY_THRESHOLD

        print(f"Inference: Max Z-score = {max_z_score:.2f}, Is Anomaly = {is_anomaly}")

        return {
            "status": "success",
            "anomaly_score": max_z_score,
            "is_anomaly": bool(is_anomaly), 
            "model_version": "z-score-v1"
        }