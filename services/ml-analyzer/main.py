from fastapi import FastAPI
from pydantic import BaseModel
from typing import List
import pandas as pd
import numpy as np

app = FastAPI()

class Metric(BaseModel):
    hostname: str
    timestamp: str
    cpu_usage_percent: float
    mem_usage_percent: float
    load_1: float
    disk_reads_ps: float
    disk_writes_ps: float
    net_sent_bytes_ps: float
    net_recv_bytes_ps: float

class InferenceRequest(BaseModel):
    window: List[Metric]

MODEL_STORE = {
    "is_trained": False,
    "training_data": pd.DataFrame(),
    "training_windows_required": 10,
    "mean": None,
    "std_dev": None
}
ANOMALY_THRESHOLD = 3.5

@app.post("/infer")
def infer_anomaly(request: InferenceRequest):
    df_window = pd.DataFrame([metric.model_dump() for metric in request.window])
    numeric_cols = [col for col in df_window.columns if df_window[col].dtype in [np.float64, np.int64]]
    if not MODEL_STORE["is_trained"]:
        MODEL_STORE["training_data"] = pd.concat([MODEL_STORE["training_data"], df_window[numeric_cols]], ignore_index=True)

        num_windows_collected = len(MODEL_STORE["training_data"]) / len(df_window)
        print(f"Collecting data for training... ({int(num_windows_collected)}/{MODEL_STORE['training_windows_required']})")

        if num_windows_collected >= MODEL_STORE["training_windows_required"]:
            MODEL_STORE["mean"] = MODEL_STORE["training_data"].mean()
            MODEL_STORE["std_dev"] = MODEL_STORE["training_data"].std()
            MODEL_STORE["is_trained"] = True
            print("--- Multi-variate Model Training Complete! ---")
            print("Mean values:\n", MODEL_STORE["mean"])
            print("\nStd Dev values:\n", MODEL_STORE["std_dev"])

        return {"status": "training", "anomaly_score": 0, "is_anomaly": False}

    else:
        std_devs = MODEL_STORE["std_dev"].replace(0, 1)
        z_scores = (df_window[numeric_cols] - MODEL_STORE["mean"]) / std_devs
        max_z_score = z_scores.abs().max().max()

        is_anomaly = max_z_score > ANOMALY_THRESHOLD
        print(f"Inference: Max Z-score across all metrics = {max_z_score:.2f}, Is Anomaly = {is_anomaly}")

        return {
            "status": "success",
            "anomaly_score": float(max_z_score),
            "is_anomaly": bool(is_anomaly),
            "model_version": "z-score-v2-multivariate"
        }