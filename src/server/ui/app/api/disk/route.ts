import { createApiResponse, createApiError } from "@/lib/api-response";
import { Disk } from "@/types";

export async function GET() {
  const getDisks = new Promise<Disk[]>(async (resolve, reject) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_SERVER_URL}/api/v1/disk`,
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
    const res = await getDisks;
    return createApiResponse(res || []);
  } catch (error) {
    console.error(error);
    return createApiError("Internal Server Error");
  }
}

export async function POST() {
  const createDisk = new Promise<Disk>(async (resolve, reject) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_SERVER_URL}/api/v1/disk`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer sk-ac-${process.env.ROOT_API_BEARER_TOKEN}`,
          },
        }
      );
      if (response.status !== 201) {
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
    const res = await createDisk;
    return createApiResponse(res);
  } catch (error) {
    console.error(error);
    return createApiError("Internal Server Error");
  }
}

