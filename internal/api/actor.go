package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/auth"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

var ErrUnauthorized = errors.New("api: unauthorized")

type ActorResolver func(*http.Request) (store.Actor, error)

type actorContextKey struct{}

// WithActor stores an already authenticated actor in the request context. An
// authentication middleware can use this hook without coupling itself to the
// game handlers.
func WithActor(r *http.Request, actor store.Actor) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), actorContextKey{}, actor))
}

// ActorMiddleware resolves an actor once and makes it available to handlers.
// NewGamesHandler also accepts a resolver directly so tests can use either
// style without installing middleware.
func ActorMiddleware(resolve ActorResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, err := resolve(r)
		if err != nil {
			writeActorError(w, err)
			return
		}
		next.ServeHTTP(w, WithActor(r, actor))
	})
}

func actorFromRequest(r *http.Request, resolve ActorResolver) (store.Actor, error) {
	if value, ok := r.Context().Value(actorContextKey{}).(store.Actor); ok {
		if strings.TrimSpace(value.ID) == "" {
			return store.Actor{}, ErrUnauthorized
		}
		return value, nil
	}
	if resolve == nil {
		return store.Actor{}, ErrUnauthorized
	}
	actor, err := resolve(r)
	if err != nil {
		return store.Actor{}, err
	}
	if strings.TrimSpace(actor.ID) == "" {
		return store.Actor{}, ErrUnauthorized
	}
	return actor, nil
}

// DevActorResolver is intentionally explicit: callers must opt into the
// development resolver when mounting the local test server. The query
// parameter is not inspected by BearerActorResolver.
func DevActorResolver(defaultPlayer string) ActorResolver {
	return func(r *http.Request) (store.Actor, error) {
		player := strings.TrimSpace(r.URL.Query().Get("player"))
		if player == "" {
			player = strings.TrimSpace(r.Header.Get("X-Dev-Player"))
		}
		if player == "" {
			player = strings.TrimSpace(defaultPlayer)
		}
		if player == "" {
			return store.Actor{}, ErrUnauthorized
		}
		return store.Actor{ID: player, Development: true}, nil
	}
}

// BearerActorResolver separates token verification from HTTP routing. O6 can
// provide a Firebase verifier without changing any game handler.
func BearerActorResolver(verify func(string) (store.Actor, error)) ActorResolver {
	return BearerActorResolverWithContext(func(_ context.Context, token string) (store.Actor, error) {
		if verify == nil {
			return store.Actor{}, ErrUnauthorized
		}
		return verify(token)
	})
}

// BearerActorResolverWithContext lets token verifiers apply request deadlines
// and cancellation while retaining the same strict Bearer parsing rules.
func BearerActorResolverWithContext(verify func(context.Context, string) (store.Actor, error)) ActorResolver {
	return func(r *http.Request) (store.Actor, error) {
		value := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(value, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return store.Actor{}, ErrUnauthorized
		}
		if verify == nil {
			return store.Actor{}, ErrUnauthorized
		}
		actor, err := verify(r.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			return store.Actor{}, fmt.Errorf("%w: %v", ErrUnauthorized, err)
		}
		if strings.TrimSpace(actor.ID) == "" {
			return store.Actor{}, ErrUnauthorized
		}
		return actor, nil
	}
}

func FirebaseActorResolver(verifier auth.Verifier) ActorResolver {
	return BearerActorResolverWithContext(func(ctx context.Context, token string) (store.Actor, error) {
		if verifier == nil {
			return store.Actor{}, ErrUnauthorized
		}
		identity, err := verifier.VerifyIDToken(ctx, token)
		if err != nil {
			return store.Actor{}, err
		}
		return store.Actor{ID: identity.UID, Email: identity.Email}, nil
	})
}

func writeActorError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrUnauthorized) {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "an authenticated actor is required")
		return
	}
	writeAPIError(w, http.StatusBadRequest, "invalid_actor", err.Error())
}
