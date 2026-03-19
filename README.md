# go-shortener

Реализация сервиса сокращения ссылок. В данной ветке реализована первая итерация задания. Примеры работы программы ниже.

![Создание ссылки](.github/images/demonstration_iter1_1.png)

![Переадресация по ссылке](.github/images/demonstration_iter1_2.png)

### Улучшение производительности

Во время реализации Инкремента №17, была улучшена производительность сервисов и
хранилища (in-memory). Результаты приведены ниже.

![Улучшение производительности сервисов](.github/images/perf_diff_services.png)

Текстовый результат изменений производительности:

```bash
pelfox@pelfox-mac ~/Programming/go-shortener (iter19) ❯ pprof -top -diff_base=profiles/base.pprof profiles/result.pprof                                                                                                       0
File: services.test
Type: alloc_space
Time: 2026-03-14 14:45:23 MSK
Showing nodes accounting for -1.47GB, 10.47% of 14.04GB total
Dropped 42 nodes (cum <= 0.07GB)
      flat  flat%   sum%        cum   cum%
    2.40GB 17.12% 17.12%     2.40GB 17.12%  github.com/Pelfox/go-shortener/internal/services.(*ShortenerService).buildShortURL (inline)
   -2.28GB 16.24%  0.88%    -2.91GB 20.76%  net/url.(*URL).joinPath
   -1.85GB 13.20% 12.33%    -1.85GB 13.20%  net/url.parse
    1.49GB 10.65%  1.68%     1.49GB 10.65%  github.com/Pelfox/go-shortener/internal/storage.(*InMemoryStorage).GetForUser
   -0.63GB  4.49%  6.17%    -0.63GB  4.49%  internal/bytealg.MakeNoZero
   -0.27GB  1.93%  8.11%    -0.27GB  1.95%  fmt.Sprintf
   -0.22GB  1.54%  9.65%    -0.22GB  1.54%  path.(*lazybuf).append (inline)
   -0.22GB  1.54% 11.19%    -0.63GB  4.52%  path.Join
    0.20GB  1.43%  9.76%    -0.63GB  4.50%  github.com/Pelfox/go-shortener/internal/services.(*ShortenerService).GetUserLinks
   -0.20GB  1.43% 11.19%    -0.20GB  1.43%  path.(*lazybuf).string (inline)
    0.17GB  1.19% 10.00%     0.14GB  0.98%  github.com/Pelfox/go-shortener/internal/services.(*UserService).CreateUserCookie
   -0.07GB  0.47% 10.47%    -0.09GB  0.67%  crypto/internal/fips140/hmac.New[go.shape.interface { BlockSize int; Reset; Size int; Sum []uint8; Write  }]
    0.06GB  0.45% 10.02%     0.06GB  0.45%  encoding/base64.(*Encoding).EncodeToString
   -0.03GB   0.2% 10.22%    -0.03GB   0.2%  crypto/internal/fips140/sha256.New (inline)
   -0.02GB  0.17% 10.39%    -0.02GB  0.17%  strings.genSplit
   -0.01GB 0.077% 10.47%    -0.05GB  0.34%  github.com/Pelfox/go-shortener/pkg.SignUserCookie
         0     0% 10.47%    -0.09GB  0.67%  crypto/hmac.New
         0     0% 10.47%    -0.03GB   0.2%  crypto/hmac.New.UnwrapNew[go.shape.interface { BlockSize int; Reset; Size int; Sum []uint8; Write  }].func1
         0     0% 10.47%    -0.03GB   0.2%  crypto/sha256.New
         0     0% 10.47%    -0.67GB  4.75%  github.com/Pelfox/go-shortener/internal/services.(*ShortenerService).CreateShortLink
         0     0% 10.47%    -0.31GB  2.23%  github.com/Pelfox/go-shortener/internal/services.(*UserService).VerifyUserCookieValue (inline)
         0     0% 10.47%    -0.67GB  4.76%  github.com/Pelfox/go-shortener/internal/services.BenchmarkCreateShortLink
         0     0% 10.47%    -0.63GB  4.50%  github.com/Pelfox/go-shortener/internal/services.BenchmarkGetUserLinks
         0     0% 10.47%     0.14GB  0.98%  github.com/Pelfox/go-shortener/internal/services.BenchmarkUserService_CreateUserCookie
         0     0% 10.47%    -0.31GB  2.23%  github.com/Pelfox/go-shortener/internal/services.BenchmarkUserService_VerifyUserCookieValue
         0     0% 10.47%    -0.31GB  2.23%  github.com/Pelfox/go-shortener/pkg.VerifyUserCookie
         0     0% 10.47%    -0.63GB  4.49%  net/url.(*URL).String
         0     0% 10.47%    -5.40GB 38.45%  net/url.JoinPath
         0     0% 10.47%    -1.85GB 13.20%  net/url.Parse
         0     0% 10.47%    -0.42GB  2.97%  path.Clean
         0     0% 10.47%    -0.63GB  4.49%  strings.(*Builder).Grow
         0     0% 10.47%    -0.63GB  4.49%  strings.(*Builder).grow
         0     0% 10.47%    -0.02GB  0.17%  strings.SplitN (inline)
         0     0% 10.47%    -1.48GB 10.55%  testing.(*B).launch
         0     0% 10.47%    -1.48GB 10.54%  testing.(*B).runN
```
