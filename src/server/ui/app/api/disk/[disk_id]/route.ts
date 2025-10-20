import { createApiResponse, createApiError } from "@/lib/api-response";

export async function DELETE(
  req: Request,
  { params }: { params: Promise<{ disk_id: string }> }
) {
  const disk_id = (await params).disk_id;
  if (!disk_id) {
    return createApiError("disk_id is required");
  }

  const deleteDisk = new Promise<null>(async (resolve, reject) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_SERVER_URL}/api/v1/disk/${disk_id}`,
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
    await deleteDisk;
    return createApiResponse(null);
  } catch (error) {
    console.error(error);
    return createApiError("Internal Server Error");
  }
}

