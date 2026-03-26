import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import type { UpdateUserInput, UpdateInterestsInput } from '@/types/api';

export function useUserInterests(enabled: boolean = true) {
  return useQuery({
    queryKey: queryKeys.user.interests,
    queryFn: () => apiClient.users.interests(),
    enabled,
  });
}

export function useUpdateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateUserInput) => apiClient.users.update(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.user.me });
    },
  });
}

export function useUpdateInterests() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateInterestsInput) => apiClient.users.updateInterests(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.user.interests });
    },
  });
}
