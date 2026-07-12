import api from './http'

export const authApi = {
  login: (data: { username: string; password: string }) => api.post('/v1/admin/login', data),
  getProfile: () => api.get('/v1/admin/profile'),
  updateProfile: (data: any) => api.put('/v1/admin/profile', data),
  changePassword: (data: { oldPassword: string; newPassword: string }) => api.put('/v1/admin/password', data),
  updateLanguage: (language: string) => api.put('/v1/admin/language', { language }),
  updateSkin: (skin: string) => api.put('/v1/admin/skin', { skin })
}

export default authApi