# Throtto - Go Dynamic Rate Limitter

## How to Use

```
limiter := throtto.New(&throtto.Config{
    CapConfidence:    300, //initial likelyhood of rps which can be served optimally by the app
    Flux:             0.001, //nominal of weight transfer, the bigger the value the increased / decreased window will be faster
    MaxIncementRate:  5, //maximum number of serving capacity increment for each success request
    MaxDecrementRate: 2, //maximum number of serving capacity decrement for each failed request (faster than increment)
    Debug:            true, //show debug info
    RejectFunc:       nil, //handler function when request is rejected due insufficient capacity
})
```

i.e we are going to use gorilla mux, or another router library
```
r := http.NewServeMux()
r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    //do something
})
http.ListenAndServe(":8000", limiter.ProtectOverRequest(r))
```

## Contributor

1. @fgarnadi
2. @syariatifaris