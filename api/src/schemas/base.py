from pydantic import BaseModel, ConfigDict

class ProtectedSchema(BaseModel):
    """
    Base schema that forbids extra fields (Auto Expose).
    Ensures that only strictly defined fields are accepted in request bodies.
    """
    model_config = ConfigDict(extra='forbid')
