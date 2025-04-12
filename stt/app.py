from flask import Flask, request
import whisper
import time
import warnings
warnings.filterwarnings("ignore", message="FP16 is not supported on CPU; using FP32 instead")
import os

app = Flask(__name__)

modelName = os.getenv("STT_MODEL", "small")
model = whisper.load_model(modelName)
print(f"modelName: {modelName}", flush=True)
print(f"available models: {whisper.available_models()}", flush=True)

@app.route("/stt", methods=["POST"])
def transcribe():
    print("Received request", flush=True)
    
    if "file" not in request.files:
        return {"error": "No file provided"}, 400

    audio_file = request.files["file"]
    temp_path = audio_file.filename
    audio_file.save(temp_path)

    start_time = time.time()
    result = model.transcribe(temp_path, language="ru")
    end_time = time.time()

    duration = end_time - start_time
    print(f"Transcription took {duration:.2f} seconds", flush=True)
    print(result["text"], flush=True)
    
    return {"text": result["text"], "duration": duration}