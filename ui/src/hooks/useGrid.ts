import client from "../api/client.ts";

const useGrid = () => {
    const {data} = client.useSuspenseQuery("get", "/grid");

    return {data};
}

export default useGrid;
