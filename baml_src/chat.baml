class Message {
  role "user" | "assistant"
  content string
}

// Default client — overridden at runtime via ClientRegistry.
client<llm> DefaultLLM {
  provider "openai"
  options {
    model "gpt-4o"
    api_key env.OPENAI_API_KEY
  }
}

function Chat(messages: Message[], system_prompt: string) -> string {
  client DefaultLLM
  prompt #"
    {{ _.role("system") }}
    {{ system_prompt }}

    {{ ctx.output_format }}

    {% for m in messages %}
    {{ _.role(m.role) }}
    {{ m.content }}
    {% endfor %}
  "#
}
