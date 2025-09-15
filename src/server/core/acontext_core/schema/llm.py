from pydantic import BaseModel
from typing import Literal, Optional


class LLMFunction(BaseModel):
    name: str
    arguments: Optional[str] = None


class LLMToolCall(BaseModel):
    id: str
    function: Optional[LLMFunction] = None
    type: Literal["function", "tool"]


class LLMResponse(BaseModel):
    role: Literal["user", "assistant", "system", "tool"]

    content: Optional[str] = None
    json_content: Optional[dict] = None
    function_call: Optional[LLMToolCall] = None
    tool_calls: Optional[list[LLMToolCall]] = None
