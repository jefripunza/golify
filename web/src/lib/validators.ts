// Zod validation schemas — shared by Login + Onboarding.
import { z } from 'zod'

export const emailSchema = z
  .string()
  .trim()
  .min(1, 'Email wajib diisi')
  .email('Format email tidak valid')

// Strong password policy:
// - min 8 chars
// - at least one lowercase, one uppercase, one digit, one symbol
export const strongPasswordSchema = z
  .string()
  .min(8, 'Password minimal 8 karakter')
  .regex(/[a-z]/, 'Harus ada huruf kecil (a-z)')
  .regex(/[A-Z]/, 'Harus ada huruf besar (A-Z)')
  .regex(/[0-9]/, 'Harus ada angka (0-9)')
  .regex(/[^a-zA-Z0-9]/, 'Harus ada simbol (mis. !@#$%)')

export const loginSchema = z.object({
  email: emailSchema,
  password: z.string().min(1, 'Password wajib diisi'),
})

export const onboardSchema = z
  .object({
    email: emailSchema,
    password: strongPasswordSchema,
    confirm: z.string(),
  })
  .refine((d) => d.password === d.confirm, {
    message: 'Password tidak sama dengan konfirmasi',
    path: ['confirm'],
  })

export type LoginValues = z.infer<typeof loginSchema>
export type OnboardValues = z.infer<typeof onboardSchema>
