import axios from "axios";

const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080",
  withCredentials: true,
  timeout: 30000,
  headers: {
    "Content-Type": "application/json",
  },
});

apiClient.interceptors.response.use(
  (response) => {
    const { code, message, data } = response.data;
    if (code !== 0) {
      return Promise.reject(new Error(message || "Request failed"));
    }
    return { ...response, data };
  },
  (error) => Promise.reject(error)
);

export default apiClient;
