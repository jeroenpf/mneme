// RefreshOptions is the shared contract for a data composable's refresh().
//
// A *silent* refresh swaps data in place without toggling `loading` or
// blanking the view on failure — used by live updates so the content isn't
// torn down and rebuilt on every agent write (the reader keeps their scroll
// position and only the changed row/block flashes). Best-effort: a failed
// silent refetch keeps the current data rather than surfacing an error.
export interface RefreshOptions {
  silent?: boolean
}
