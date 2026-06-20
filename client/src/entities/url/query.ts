import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getUrls, shortenUrl } from "./api";

export function useUrls() {
  return useQuery({
    queryKey: ["urls"],
    queryFn: getUrls,
    select: (data) => data ?? [],
  });
}

export function useShortenUrl() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: shortenUrl,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["urls"] });
    },
  });
}
