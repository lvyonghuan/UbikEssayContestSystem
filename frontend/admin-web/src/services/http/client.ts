import axios from 'axios'
import { appConfig } from '@/config/env'

export const adminClient = axios.create({
  baseURL: appConfig.adminBaseURL,
  timeout: appConfig.requestTimeout,
})

export const systemClient = axios.create({
  baseURL: appConfig.systemBaseURL,
  timeout: appConfig.requestTimeout,
})
