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

// ── Domain validation ─────────────────────────────────────────────────────
// Accepts domain or subdomain. Auto-strips http:// / https:// scheme.
// Rejects paths, ports, query strings, whitespace, and invalid characters.
export const domainSchema = z
  .string()
  .trim()
  .min(1, 'Domain wajib diisi')
  .transform((raw) => {
    // strip scheme if present
    let d = raw.trim()
    if (d.includes('://')) d = d.slice(d.indexOf('://') + 3)
    // strip trailing path/query/fragment
    d = d.replace(/[/?#].*$/, '')
    d = d.replace(/\.+$/, '').toLowerCase()
    return d
  })
  .refine((d) => d.length > 0, { message: 'Domain wajib diisi' })
  .refine((d) => /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$/.test(d), {
    message: 'Format domain tidak valid (contoh: example.com atau sub.example.com)',
  })
  .refine((d) => !d.includes('..'), { message: 'Format domain tidak valid' })
  .refine((d) => !/^-|-$/.test(d.split('.')[0]), { message: 'Label domain tidak boleh diawali/diakhiri "-"' })

export type DomainValues = z.infer<typeof domainSchema>
