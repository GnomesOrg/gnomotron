from flask import Flask, request
import whisper
import os

app = Flask(__name__)

modelName = os.getenv("STT_MODEL", "small")
model = whisper.load_model(modelName)

@app.route("/stt", methods=["POST"])
def transcribe():
    print("Received request")
    
    if "file" not in request.files:
        return {"error": "No file provided"}, 400

    audio_file = request.files["file"]
    print(audio_file)

    temp_path = "temp_audio.ogg"
    audio_file.save(temp_path)

    result = model.transcribe(temp_path)
    
    return {"text": result["text"]}

if __name__ == "__main__":
    app.run(host="stt", port=5000, debug=True)
