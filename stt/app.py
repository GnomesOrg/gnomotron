from flask import Flask, request
import whisper
import os

app = Flask(__name__)

modelName = os.getenv("STT_MODEL", "small")
model = whisper.load_model(modelName)

@app.route("/stt", methods=["POST"])
def transcribe():
    print("Received request", flush=True)
    
    if "file" not in request.files:
        return {"error": "No file provided"}, 400

    audio_file = request.files["file"]

    temp_path = audio_file.filename
    audio_file.save(temp_path)

    result = model.transcribe(temp_path, language="ru")
    print(result["text"], flush=True)
    
    return {"text": result["text"]}

if __name__ == "__main__":
    app.run(host="stt", port=5000, debug=True)
