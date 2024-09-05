from string import Template
import requests
import json
from config import GPT_OAUTH

BODY_JSON = {
  "modelUri": "gpt://b1gh0fnleoo9l8ktrl5u/yandexgpt/latest",
  "completionOptions": {
    "stream": False,
    "temperature": 0.8,
    "maxTokens": "2000"
  },
  "messages": [
    {
      "role": "system",
      "text": "Отвечай от первого лица, как будто мы ведем диалог. \
      Ты мудрый гномик. Ты учавствуешь в беседе среди других гномов. Контекст разговора тебе не известен. \
      Тебе нужно отвечать на вопросы используюя сказочные термины. Ты дружелюбный гном. \
      Отвечай ОЧЕНЬ КРАТКО В ОДНО ПРЕДЛОЖЕНИЕ"
    },
  ]
}


def get_gpt_only_text(auth_headers, request_text):
    url = 'https://llm.api.cloud.yandex.net/foundationModels/v1/completion'

    data = BODY_JSON
    data["messages"].append({"role": "user", "text": request_text})
    data = json.dumps(data)

    resp = requests.post(url, headers=auth_headers, data=data)

    if resp.status_code != 200:
        raise RuntimeError(
            'Invalid response received: code: {}, message: {}'.format(
                {resp.status_code}, {resp.text}
            )
        )

    return json.loads(resp.text)['result']['alternatives'][0]['message']['text']


def get_iam_token():
    headers = {
        'Content-Type': 'application/x-www-form-urlencoded',
    }

    s = Template('{"yandexPassportOauthToken":"$GPT_OAUTH"}')
    data = s.substitute(GPT_OAUTH=GPT_OAUTH)

    response = requests.post('https://iam.api.cloud.yandex.net/iam/v1/tokens', headers=headers, data=data).json()
    return response['iamToken']


def get_gpt_response_with_message(message):
    iam_token = get_iam_token()

    headers = {
        'Authorization': f'Bearer {iam_token}',
    }

    return get_gpt_only_text(headers, message)
