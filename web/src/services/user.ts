import request from './api'
import type { ApiResponse, User, PaginatedResponse } from '../types'

export interface UserListRequest {
  page?: number
  page_size?: number
  keyword?: string
  role?: string
  status?: string
}

export interface CreateUserRequest {
  username: string
  password: string
  email: string
  phone?: string
  role: string
  status: string
}

export interface UpdateUserRequest {
  email?: string
  phone?: string
  status?: string
}

export const userApi = {
  getUsers: (params: UserListRequest = {}): Promise<ApiResponse<PaginatedResponse<User>>> => {
    return request.get('/users', { params })
  },

  getUserById: (id: number): Promise<ApiResponse<User>> => {
    return request.get(`/users/${id}`)
  },

  createUser: (data: CreateUserRequest): Promise<ApiResponse<User>> => {
    return request.post('/users', data)
  },

  updateUser: (id: number, data: UpdateUserRequest): Promise<ApiResponse<User>> => {
    return request.put(`/users/${id}`, data)
  },

  updateUserRole: (id: number, role: string): Promise<ApiResponse<User>> => {
    return request.put(`/users/${id}/role`, { role })
  },

  updateUserStatus: (id: number, status: string): Promise<ApiResponse<User>> => {
    return request.put(`/users/${id}/status`, { status })
  },

  deleteUser: (id: number): Promise<ApiResponse> => {
    return request.delete(`/users/${id}`)
  },

  resetPassword: (id: number, data: { new_password: string }): Promise<ApiResponse> => {
    return request.post(`/users/${id}/reset-password`, data)
  }
}
