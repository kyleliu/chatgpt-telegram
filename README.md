# chatgpt-telegram

2023年3月1日，OpenAI公布了最新模型`gpt-3.5-turbo`，该模型和[ChatGPT](https://openai.com/blog/chatgpt/)使用的模型是一致的。本bot是在此模型之上编写的类似ChatGPT的工具，并根据telegram对话模式做了一些改进。

## 安装
在 [Releases](https://github.com/kyleliu/chatgpt-telegram/releases/latest) 页面中下载与您的操作系统相对应的文件。

- `chatgpt-telegram-Darwin-amd64`: macOS (Intel)
- `chatgpt-telegram-Darwin-arm64`: macOS (M1)
- `chatgpt-telegram-Linux-amd64`: Linux
- `chatgpt-telegram-Linux-arm64`: Linux (ARM)
- `chatgpt-telegram-Win-amd64`: Windows

下载文件后，将其解压缩到一个文件夹中，并使用文本编辑器打开 `env.example` 文件并填写您的tokens。

- `TELEGRAM_TOKEN`: 您的Telegram Bot令牌
  - 参考此 [指南](https://core.telegram.org/bots/tutorial#obtain-your-bot-token) 创建一个机器人并获取令牌。
- `OPENAI_API_KEY`: 您在OpenAI处申请的API调用令牌
  - 参考此 [指南](https://platform.openai.com/docs/quickstart/add-your-api-key)。
- `TELEGRAM_ID` (可选): 您的Telegram用户ID
  - 如果设置了此项，则只有您可以与机器人进行交互。
  - 要获取您的ID，请在Telegram上向 `@userinfobot` 发送消息。
  - 可以提供多个ID，用逗号分隔。
- `EDIT_WAIT_SECONDS` (可选): 消息输入之间等待的秒数
  - 默认设置为`1`，但如果开始出现大量`Too Many Requests`错误，可以增加此值。
- `PROMPT_INIT` (可选): 对此模型的最高指示
  - 比如，你可以设定模型的身份：`你是一个全能助手，你的名字叫多多。`
  - 它就会以多多这个身份跟你交流。
- 保存文件，并将其重命名为`.env`。
> **注意** 一定要将文件重命名为确切的`.env`！否则程序将无法正常工作。

最后，在您的计算机上打开终端（如果您使用的是Windows，请查找`PowerShell`），导航到您提取上述文件的路径（您可以使用`cd dirname`导航到一个目录，如果需要更多帮助，可以问ChatGPT 😉），并运行`./chatgpt-telegram`。

### 在`Docker`里运行

如果你想在具有现有Docker设置的服务器上运行此程序，那么你可能需要使用我们的Docker镜像。

```sh
docker pull ghcr.io/kyleliu/chatgpt-telegram
```

如下为`docker-compose`设置:

```yaml
services:
  chatgpt-telegram:
    image: ghcr.io/kyleliu/chatgpt-telegram
    container_name: chatgpt-telegram
    volumes:
      # your ".config" local folder must include a "chatgpt.json" file
      - .config/:/root/.config
    environment:
      - TELEGRAM_ID=
      - TELEGRAM_TOKEN=
      - OPENAI_API_KEY=
      - PROMPT_INIT=
```

## 许可证

此项目来源于[m1guelpf/chatgpt-telegram](https://github.com/m1guelpf/chatgpt-telegram)，遵循[MIT许可证](LICENSE)。
