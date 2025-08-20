// File: src/services/api.ts
import axios from 'axios';

// Buat instance Axios dengan konfigurasi dasar
const apiClient = axios.create({
  //baseURL: 'http://localhost:8080/api/v1', // URL dasar backend kita
  baseURL: '/api/v1',
});

// Ini adalah "interceptor", sebuah fungsi yang akan dijalankan SEBELUM setiap request dikirim.
apiClient.interceptors.request.use(
  (config) => {
    // Ambil token dari localStorage
    const token = localStorage.getItem('authToken');
    // Jika token ada, tambahkan ke header Authorization
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

export default apiClient;