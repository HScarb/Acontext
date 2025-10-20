import { createApiResponse, createApiError } from "@/lib/api-response";
import { ListArtifactsResp } from "@/types";

export async function GET(
  req: Request,
  { params }: { params: Promise<{ disk_id: string }> }
) {
  const disk_id = (await params).disk_id;
  if (!disk_id) {
    return createApiError("disk_id is required");
  }

  const { searchParams } = new URL(req.url);
  const path = searchParams.get("path") || "/";

  const getListArtifacts = new Promise<ListArtifactsResp>(async (resolve, reject) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_SERVER_URL}/api/v1/disk/${disk_id}/artifact/ls?path=${path}`,
        {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer sk-ac-${process.env.ROOT_API_BEARER_TOKEN}`,
          },
        }
      );
      if (response.status !== 200) {
        reject(new Error("Internal Server Error"));
      }

      const result = await response.json();
      if (result.code !== 0) {
        reject(new Error(result.message));
      }
      resolve(result.data);
    } catch {
      reject(new Error("Internal Server Error"));
    }
  });

  try {
    const res = await getListArtifacts;
    return createApiResponse(res || []);
  } catch (error) {
    console.error(error);
    return createApiError("Internal Server Error");
  }
}

