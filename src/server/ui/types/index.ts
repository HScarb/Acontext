export interface Disk {
  id: string;
  project_id: string;
  created_at: string;
  updated_at: string;
}

export interface Artifact {
  disk_id: string;
  path: string;
  filename: string;
  meta: {
    __artifact_info__: {
      filename: string;
      mime: string;
      path: string;
      size: number;
    };
    [key: string]: unknown;
  };
  created_at: string;
  updated_at: string;
}

export interface ListArtifactsResp {
  artifacts: Artifact[];
  directories: string[];
}

export interface GetArtifactResp {
  artifact: Artifact;
  public_url: string | null;
}

export interface Space {
  id: string;
  project_id: string;
  configs: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Session {
  id: string;
  project_id: string;
  space_id: string | null;
  configs: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Part {
  type: string;
  text?: string;
  asset?: {
    bucket: string;
    s3_key: string;
    etag: string;
    sha256: string;
    mime: string;
    size_b: number;
  };
  filename?: string;
  meta?: Record<string, unknown>;
}

export interface Message {
  id: string;
  session_id: string;
  parent_id: string | null;
  role: string;
  parts: Part[];
  session_task_process_status: string;
  created_at: string;
  updated_at: string;
}

export interface GetMessagesResp {
  items: Message[];
  next_cursor?: string;
  has_more: boolean;
  public_urls?: Record<string, { url: string; expire_at: string }>;
}

export interface Block {
  id: string;
  space_id: string;
  type: string;
  parent_id: string | null;
  title: string;
  props: Record<string, unknown>;
  sort: number;
  is_archived: boolean;
  created_at: string;
  updated_at: string;
}

// Message related types
export type MessageRole = "user" | "assistant" | "system";

export type PartType =
  | "text"
  | "image"
  | "audio"
  | "video"
  | "file"
  | "tool-call"
  | "tool-result"
  | "data";

export interface UploadedFile {
  id: string;
  file: globalThis.File; // Browser File API
  type: PartType;
}

export interface ToolCall {
  id: string;
  tool_name: string;
  tool_call_id: string;
  parameters: string;
}

export interface ToolResult {
  id: string;
  tool_call_id: string;
  result: string;
}
