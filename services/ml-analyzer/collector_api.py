from fastapi import FastAPI
from pydantic import BaseModel
from typing import List
import pandas as pd
import numpy as np
import os

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

DATA_COLLECTION_STORE = {
    "all_data": [],
    "windows_collected": 0,
    "windows_to_collect": 200,
    "is_complete": False,
    "output_path": "training_data.csv"
}

@app.post("/infer")
def collect_data(request: InferenceRequest):
    if DATA_COLLECTION_STORE["is_complete"]:
        return {"status": "Data collection is complete. Please train the model."}
    for metric in request.window:
        DATA_COLLECTION_STORE["all_data"].append(metric.model_dump())

    DATA_COLLECTION_STORE["windows_collected"] += 1

    print(f"Collecting data for training... ({DATA_COLLECTION_STORE['windows_collected']}/{DATA_COLLECTION_STORE['windows_to_collect']})")
    if DATA_COLLECTION_STORE["windows_collected"] >= DATA_COLLECTION_STORE["windows_to_collect"]:
        df = pd.DataFrame(DATA_COLLECTION_STORE["all_data"])
        df.to_csv(DATA_COLLECTION_STORE["output_path"], index=False)
        DATA_COLLECTION_STORE["is_complete"] = True
        print(f"--- Data collection complete! Saved to {DATA_COLLECTION_STORE['output_path']} ---")

    return {"status": "collecting_data"}