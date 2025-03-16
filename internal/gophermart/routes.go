package gophermart

import (
	"net/http"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/handlers"
	"github.com/gorilla/mux"
)

// SetupRouter настраивает маршрутизацию для API
func SetupRouter(h *handlers.Handler, authService *auth.Auth) *mux.Router {
	r := mux.NewRouter()

	// Создаем подмаршрутизатор для публичных маршрутов
	public := r.PathPrefix("/api").Subrouter()

	// Регистрируем маршруты для аутентификации
	public.HandleFunc("/user/register", h.Register).Methods(http.MethodPost)
	public.HandleFunc("/user/login", h.Login).Methods(http.MethodPost)

	// Создаем подмаршрутизатор для защищенных маршрутов
	private := r.PathPrefix("/api").Subrouter()
	private.Use(authService.AuthMiddleware)

	// Регистрируем защищенные маршруты для работы с заказами
	private.HandleFunc("/user/orders", h.CreateOrder).Methods(http.MethodPost)
	private.HandleFunc("/user/orders", h.GetOrders).Methods(http.MethodGet)

	// Регистрируем защищенные маршруты для работы с балансом
	private.HandleFunc("/user/balance", h.GetBalance).Methods(http.MethodGet)
	private.HandleFunc("/user/balance/add", h.AddBalance).Methods(http.MethodPost)
	private.HandleFunc("/user/balance/withdraw", h.Withdraw).Methods(http.MethodPost)
	private.HandleFunc("/user/withdrawals", h.GetWithdrawals).Methods(http.MethodGet)

	return r
}
