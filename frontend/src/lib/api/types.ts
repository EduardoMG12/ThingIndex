export type ApiSuccessResponse<T> = {
    status: number;
    type: string;
    message: string;
    data: T;
};

export type ApiErrorResponse = {
    status: number;
    type: string;
    message: string;
    errors?: unknown;
};

export function hasResponseData<T>(
    payload: ApiSuccessResponse<T> | ApiErrorResponse | null
): payload is ApiSuccessResponse<T> {
    return Boolean(payload && "data" in payload);
}
