#!/bin/sh

MODEL_FILE="autoencoder_model.pth"
DATA_FILE="training_data.csv"
SCALER_FILE="scaler.joblib"
THRESHOLD_FILE="threshold.txt"

if [ -f "$MODEL_FILE" ] && [ -f "$SCALER_FILE" ] && [ -f "$THRESHOLD_FILE" ]; then
    echo "--- Model, scaler, and threshold found! Starting inference server. ---"
    uvicorn main:app --host 0.0.0.0 --port 8000
else
    echo "--- One or more artifacts not found. Starting first-time setup... ---"

    if [ ! -f "$DATA_FILE" ]; then
        echo "--> Training data not found. Starting data collection..."
        uvicorn collector_api:app --host 0.0.0.0 --port 8000 &
        PID=$!
        while [ ! -f "$DATA_FILE" ]; do sleep 10; done
        echo "--> Data collection complete. Stopping collector API."
        kill $PID
        sleep 5
    else
        echo "--> Found existing training data. Skipping collection."
    fi

    echo "--> Starting model training..."
    python train.py

    echo "--- Setup complete! Starting final inference server. ---"
    uvicorn main:app --host 0.0.0.0 --port 8000
fi