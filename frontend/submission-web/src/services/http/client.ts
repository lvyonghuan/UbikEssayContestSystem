import axios from 'axios'
import { appConfig } from '@/config/env'

export const submissionClient = axios.create({
  baseURL: appConfig.submissionBaseURL,
  timeout: appConfig.requestTimeout,
})

export const systemClient = axios.create({
  baseURL: appConfig.systemBaseURL,
  timeout: appConfig.requestTimeout,
})
