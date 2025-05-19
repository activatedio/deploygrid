import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";
import {paths} from "./schema";

const API_URL = import.meta.env.VITE_API_URL;


const fetchClient = createFetchClient<paths>({
    baseUrl: API_URL,
});
const client = createClient(fetchClient);

export default client;
