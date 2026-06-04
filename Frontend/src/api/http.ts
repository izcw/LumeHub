import axios from 'axios'

/** 生产环境由 Go 同域托管时留空；独立域名部署时通过 VITE_API_BASE 指定 */
const baseURL = import.meta.env.VITE_API_BASE ?? ''

export const http = axios.create({
  baseURL,
  timeout: 60_000,
  withCredentials: true,
})
