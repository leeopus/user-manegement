const API_BASE = process.env.NEXT_PUBLIC_API_URL || ""

export interface LoginRequest {
  email: string
  password: string
}

export interface User {
  ID: number
  Email: string
  Username: string
  Avatar?: string
  Status?: string
}

export interface LoginResponseData {
  user: User
  access_token: string
  refresh_token: string
}

export interface LoginResponse {
  code: number
  message: string
  data?: LoginResponseData
}

export interface RegisterRequest {
  email: string
  password: string
  username?: string
}

class APIClient {
  private baseURL: string

  constructor(baseURL: string = API_BASE) {
    this.baseURL = baseURL
  }

  private getUrl(path: string): string {
    // If baseURL is set, use it (for production)
    // Otherwise use relative path (for development with Next.js rewrites)
    return this.baseURL ? `${this.baseURL}${path}` : path
  }

  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await fetch(this.getUrl("/api/v1/auth/login"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(credentials),
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    return response.json()
  }

  async register(data: RegisterRequest): Promise<LoginResponse> {
    const response = await fetch(this.getUrl("/api/v1/auth/register"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    return response.json()
  }

  async getUserInfo(token: string): Promise<{ code: number; data?: User }> {
    const response = await fetch(this.getUrl("/api/v1/auth/me"), {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    return response.json()
  }
}

export const api = new APIClient()
