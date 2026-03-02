"use server";

export async function fetchGraphTopics() {
  try {
    // In Docker, 'api' is the hostname of the backend container. 
    // NEXT_PUBLIC_API_URL is meant for the browser, so we use an internal URL fallback.
    const baseUrl = process.env.API_INTERNAL_URL || "http://api:8000/api/v1";
    const res = await fetch(`${baseUrl}/graph/topics`, {
      cache: "no-store", // Or revalidate if you want caching
    });
    
    if (!res.ok) {
      throw new Error(`Failed to fetch topics: ${res.statusText}`);
    }
    
    const data = await res.json();
    return data.topics || [];
  } catch (error) {
    console.error("Error fetching graph topics:", error);
    return [];
  }
}

export async function fetchGraphData(context: string = "all", limit: number = 3000) {
  try {
    const baseUrl = process.env.API_INTERNAL_URL || "http://api:8000/api/v1";
    const res = await fetch(
      `${baseUrl}/graph/visualize?context=${encodeURIComponent(context)}&limit=${limit}`,
      {
        cache: "no-store",
      }
    );

    if (!res.ok) {
      throw new Error(`Failed to fetch graph data: ${res.statusText}`);
    }

    const data = await res.json();
    return {
      nodes: data.nodes || [],
      links: data.links || [],
      totalNodes: data.total_nodes || 0
    };
  } catch (error) {
    console.error("Error fetching graph data:", error);
    return { nodes: [], links: [], totalNodes: 0 };
  }
}

export async function startCrawl(url: string, dirName: string, depth: number) {
  try {
    const baseUrl = process.env.API_INTERNAL_URL?.replace("/api/v1", "") || "http://api:8000";
    const res = await fetch(`${baseUrl}/crawl`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url, dir_name: dirName, depth }),
    });

    if (!res.ok) {
      const errorData = await res.json().catch(() => null);
      throw new Error(errorData?.detail || res.statusText);
    }

    return await res.json();
  } catch (error) {
    console.error("Error starting crawl:", error);
    return { status: "error", message: String(error) };
  }
}

export async function purgeCrawler() {
  try {
    const baseUrl = process.env.API_INTERNAL_URL?.replace("/api/v1", "") || "http://api:8000";
    const res = await fetch(`${baseUrl}/purge`, {
      method: "POST",
    });

    if (!res.ok) {
      throw new Error(res.statusText);
    }

    return await res.json();
  } catch (error) {
    console.error("Error purging crawler:", error);
    return { status: "error", message: String(error) };
  }
}
