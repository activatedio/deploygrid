import client from "../api/client.ts";

const useGrid = (system: string) => {
    const {data} = client.useSuspenseQuery("get", "/systems/{system}/grid", {
        params: {path: {system}},
    }, {
        refetchInterval: 15_000,
    });

    return {data};
}

export default useGrid;
