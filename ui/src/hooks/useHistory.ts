import client from "../api/client.ts";

const useHistory = (system: string, component: string) => {
    const {data} = client.useSuspenseQuery("get", "/systems/{system}/components/{component}/history", {
        params: {path: {system, component}},
    }, {
        refetchInterval: 15_000,
    });
    return {changes: data.items ?? []};
}

export default useHistory;
