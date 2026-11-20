export type User = {
  id: string
  name: string
  login_name: string
  active: boolean
}

export type Credentials = {
  login_name: string
  password: string
}

export type Registration = Credentials & {
  name: string
}
