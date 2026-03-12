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
