import client from "../api/client.ts";

const useSystems = () => {
    const {data} = client.useSuspenseQuery("get", "/systems");
    return {systems: data.items ?? []};
}

export default useSystems;
