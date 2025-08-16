import torch
import pandas as pd
from fastapi import FastAPI
from pydantic import BaseModel
from typing import List
import joblib
from model import Autoencoder
from sklearn.preprocessing import MinMaxScaler

MODEL_PATH = "autoencoder_model.pth"
SCALER_PATH = "scaler.joblib"
THRESHOLD_PATH = "threshold.txt"
N_FEATURES = 7

app = FastAPI(title="Anomaly Detection Inference Server")

try:
    model = Autoencoder(n_features=N_FEATURES)
    model.load_state_dict(torch.load(MODEL_PATH))
    model.eval()
    print("--- Autoencoder model loaded successfully! ---")

    scaler = joblib.load(SCALER_PATH)
    print("--- Data scaler loaded successfully! ---")

    with open(THRESHOLD_PATH, "r") as f:
        ANOMALY_THRESHOLD = float(f.read())
    print(f"--- Anomaly threshold loaded: {ANOMALY_THRESHOLD:.6f} ---")

except FileNotFoundError as e:
    print(f"--- FATAL ERROR: Model, scaler, or threshold file not found. {e} ---")
    model = None 
    scaler = None
    ANOMALY_THRESHOLD = 999

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

@app.post("/infer")
def infer_anomaly(request: InferenceRequest):
    if not model or not scaler:
        return {"status": "error", "message": "Model or scaler not loaded."}

    df_window = pd.DataFrame([metric.model_dump() for metric in request.window])
    numeric_cols = [col for col in df_window.columns if df_window[col].dtype in ['float64', 'int64']]

    window_scaled = scaler.transform(df_window[numeric_cols])
    window_tensor = torch.FloatTensor(window_scaled)

    with torch.no_grad():
        reconstructions = model(window_tensor)

    mse_loss = torch.mean((window_tensor - reconstructions) ** 2, dim=1)
    max_error = torch.max(mse_loss).item()
    is_anomaly = max_error > ANOMALY_THRESHOLD

    print(f"Inference: Max reconstruction error = {max_error:.6f}, Is Anomaly = {is_anomaly}")

    return {
        "status": "success",
        "anomaly_score": float(max_error),
        "is_anomaly": bool(is_anomaly),
        "model_version": "autoencoder-v3-final"
    }