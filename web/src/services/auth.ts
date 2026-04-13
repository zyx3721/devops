import request from './api'
import type { ApiResponse, User } from '../types'

export interface LoginResponse {
  token: string
  user: User
}

export const authApi = {
  login: (username: string, password: string): Promise<ApiResponse<LoginResponse>> => {
    return request.post('/auth/login', { username, password })
  },

  register: (data: { username: string; password: string; email: string; phone?: string }): Promise<ApiResponse<User>> => {
    return request.post('/auth/register', data)
  },

  getProfile: (): Promise<ApiResponse<User>> => {
    return request.get('/users/profile')
  },

  changePassword: (data: { old_password: string; new_password: string }): Promise<ApiResponse> => {
    return request.post('/users/change-password', data)
  }
}
