import service, { Res } from "../http";
import { Artifact, ListFilesResp, GetFileResp } from "@/types";

export const getArtifacts = async (): Promise<Res<Artifact[]>> => {
  return await service.get("/api/artifact");
};

export const getListFiles = async (
  artifact_id: string,
  path: string
): Promise<Res<ListFilesResp>> => {
  return await service.get(`/api/artifact/${artifact_id}/file/ls?path=${path}`);
};

export const getFile = async (
  artifact_id: string,
  file_path: string
): Promise<Res<GetFileResp>> => {
  return await service.get(
    `/api/artifact/${artifact_id}/file?file_path=${file_path}`
  );
};

export const createArtifact = async (): Promise<Res<Artifact>> => {
  return await service.post("/api/artifact");
};

export const deleteArtifact = async (
  artifact_id: string
): Promise<Res<null>> => {
  return await service.delete(`/api/artifact/${artifact_id}`);
};
