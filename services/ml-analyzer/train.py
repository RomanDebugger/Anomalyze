import torch
from torch import nn
import pandas as pd
from sklearn.preprocessing import MinMaxScaler
from model import Autoencoder
import joblib 
import argparse
import sys
import numpy as np

parser = argparse.ArgumentParser(description='Train Autoencoder Model')
parser.add_argument('--data-path', type=str, default='training_data.csv', help='Path to the training data CSV file')
parser.add_argument('--model-path', type=str, default='autoencoder_model.pth', help='Path to save the trained model')
parser.add_argument('--scaler-path', type=str, default='scaler.joblib', help='Path to save the data scaler')
parser.add_argument('--epochs', type=int, default=50, help='Number of training epochs')
parser.add_argument('--n-features', type=int, default=7, help='Number of features for the model')
args = parser.parse_args()

print("Loading and preprocessing data...")
try:
    df = pd.read_csv(args.data_path)
except FileNotFoundError:
    print(f"Error: Training data file not found at {args.data_path}")
    sys.exit(1)

numeric_cols = [col for col in df.columns if df[col].dtype in ['float64', 'int64']]
if len(numeric_cols) != args.n_features:
    print(f"Error: Expected {args.n_features} numeric features, but found {len(numeric_cols)}.")
    sys.exit(1)

df_numeric = df[numeric_cols]

scaler = MinMaxScaler()
data_scaled = scaler.fit_transform(df_numeric)
train_tensor = torch.FloatTensor(data_scaled)

model = Autoencoder(n_features=args.n_features)
criterion = nn.MSELoss()
optimizer = torch.optim.Adam(model.parameters(), lr=1e-3)

print("Starting model training...")
for epoch in range(args.epochs):
    outputs = model(train_tensor)
    loss = criterion(outputs, train_tensor)

    optimizer.zero_grad()
    loss.backward()
    optimizer.step()

    if (epoch+1) % 10 == 0:
        print(f'Epoch [{epoch+1}/{args.epochs}], Loss: {loss.item():.6f}')

torch.save(model.state_dict(), args.model_path)
print(f"--- Training Complete! Model saved to {args.model_path} ---")

joblib.dump(scaler, args.scaler_path)
print(f"--- Scaler saved to {args.scaler_path} ---")

print("\n--- Calculating anomaly threshold ---")
with torch.no_grad():
    reconstructions = model(train_tensor)
    # Calculate loss for each data point
    reconstruction_loss = torch.mean((train_tensor - reconstructions) ** 2, dim=1)

    # Set threshold at the 99th percentile
    threshold = np.percentile(reconstruction_loss.numpy(), 99)
    print(f"Anomaly threshold (99th percentile): {threshold:.6f}")
    with open("threshold.txt", "w") as f:
        f.write(str(threshold))
    print("--- Threshold saved to threshold.txt ---")