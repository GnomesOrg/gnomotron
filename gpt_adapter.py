import requests
import json
from config import API_KEY
from copy import deepcopy

BODY_JSON = {
  "modelUri": "gpt://b1gh0fnleoo9l8ktrl5u/yandexgpt/latest",
  "completionOptions": {
    "stream": False,
    "temperature": 1,
    "maxTokens": "500"
  },
  "messages": [
    {
      "role": "system",
      "text": "Отвечай от первого лица, как будто мы ведем диалог. Ты мудрый гномик. Ты учавствуешь в беседе среди "
              "других гномов. Тебе нужно отвечать на вопросы, используюя сказочные термины. Ты дружелюбный гном. "
              "Отвечай ОЧЕНЬ КРАТКО В ОДНО ПРЕДЛОЖЕНИЕ"
    },
  ]
}


def get_gpt_only_text(auth_headers, request_text):
    url = 'https://llm.api.cloud.yandex.net/foundationModels/v1/completion'

    data = deepcopy(BODY_JSON)
    data["messages"].append({"role": "user", "text": request_text})
    data = json.dumps(data, ensure_ascii=False)

    resp = requests.post(url, headers=auth_headers, data=data.encode('utf-8'))
    print('--- request text ' + data)
    print('--- response text ' + resp.text)

    if resp.status_code != 200:
        raise RuntimeError(
            'Invalid response received: code: {}, message: {}'.format(
                {resp.status_code}, {resp.text}
            )
        )

    return json.loads(resp.text)['result']['alternatives'][0]['message']['text']


def get_gpt_response_with_message(message):
    headers = {
        'Content-Type': "application/json",
        'Authorization': f'Api-Key {API_KEY}',
    }

    result = get_gpt_only_text(headers, message)

    return result
