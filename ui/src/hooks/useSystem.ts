import client from "../api/client.ts";

const useSystem = (system: string) => {
    const {data} = client.useSuspenseQuery("get", "/systems/{system}", {
        params: {path: {system}},
    });
    return {system: data};
}

export default useSystem;
