import client from "../api/client.ts";

const useComponent = (system: string, component: string) => {
    const {data} = client.useSuspenseQuery("get", "/systems/{system}/components/{component}", {
        params: {path: {system, component}},
    }, {
        refetchInterval: 15_000,
    });
    return {row: data};
}

export default useComponent;
