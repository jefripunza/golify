# Golify TODO

## Task
- [x] R1: Fix logout — hapus token + redirect ke login (global guard)
- [x] R2: Halaman 404 + error boundary (stack + tombol copy)
- [x] R3: BE — auth/status + auth/onboard, User pakai email (admin pertama)
- [x] R4: FE — OnboardingView wizard (welcome → admin account), Zod strong password
- [x] R5: FE — LoginView ganti username→email + Zod validasi
- [x] R6: Router — /onboarding, /login, /404 + guard onboarding
- [x] R7: WS backend integrasi + smoke test end-to-end

## Blocked
- (none)

## Finish
- [x] Smoke test BE: status → onboard (weak=400, valid=201) → status onboarded=true → re-onboard=409 → login email=200 → login username lama=401 → ws health=ok
- [x] Browser: logout → /login + token hilang; login email → dashboard
- [x] FE build + BE build hijau
