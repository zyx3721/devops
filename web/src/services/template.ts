import request from './api'
import type { ApiResponse } from '../types'

export interface MessageTemplate {
  id: number
  name: string
  template_type: string
  title: string
  content: string
  variables: string
  description?: string
  is_active: boolean
}

export const templateApi = {
  list: (params?: { keyword?: string }): Promise<ApiResponse<{ list: MessageTemplate[]; total: number }>> => {
    return request.get('/notification/templates', { params })
  },
  
  get: (id: number): Promise<ApiResponse<MessageTemplate>> => {
    return request.get(`/notification/templates/${id}`)
  },
  
  preview: (data: { template_id?: number; content?: string; data: Record<string, any> }): Promise<ApiResponse<string>> => {
    return request.post('/notification/templates/preview', data)
  }
}
