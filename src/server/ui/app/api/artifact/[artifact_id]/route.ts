import { createApiResponse, createApiError } from "@/lib/api-response";

export async function DELETE(
  req: Request,
  { params }: { params: Promise<{ artifact_id: string }> }
) {
  const artifact_id = (await params).artifact_id;
  if (!artifact_id) {
    return createApiError("artifact_id is required");
  }

  const deleteArtifact = new Promise<null>(async (resolve, reject) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_SERVER_URL}/api/v1/artifact/${artifact_id}`,
        {
          method: "DELETE",
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
      resolve(null);
    } catch {
      reject(new Error("Internal Server Error"));
    }
  });

  try {
    await deleteArtifact;
    return createApiResponse(null);
  } catch (error) {
    console.error(error);
    return createApiError("Internal Server Error");
  }
}

