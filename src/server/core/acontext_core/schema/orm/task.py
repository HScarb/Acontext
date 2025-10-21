from dataclasses import dataclass, field
from sqlalchemy import (
    ForeignKey,
    Index,
    Column,
    Integer,
    String,
    Enum,
    text,
    CheckConstraint,
    UniqueConstraint,
    Boolean,
)
from sqlalchemy.orm import relationship
from sqlalchemy.dialects.postgresql import JSONB, UUID, ARRAY
from typing import TYPE_CHECKING, Optional, List
from .base import ORM_BASE, CommonMixin
from ..session.task import TaskStatus
from ..utils import asUUID

if TYPE_CHECKING:
    from .project import Project
    from .session import Session
    from .message import Message

# TaskStatusEnum = Enum(TaskStatus, name="task_status_enum", create_type=True)


@ORM_BASE.mapped
@dataclass
class Task(CommonMixin):
    __tablename__ = "tasks"

    __table_args__ = (
        CheckConstraint(
            "status IN ('success', 'failed', 'running', 'pending')",
            name="ck_status",
        ),
        UniqueConstraint(
            "session_id",
            "order",
            name="uq_session_id_order",
        ),
        Index("ix_task_session_id", "session_id"),
        Index("ix_task_session_id_task_id", "session_id", "id"),
        Index("ix_task_session_id_status", "session_id", "status"),
        Index("ix_task_project_id", "project_id"),
    )

    session_id: asUUID = field(
        metadata={
            "db": Column(
                UUID(as_uuid=True),
                ForeignKey("sessions.id", ondelete="CASCADE"),
                nullable=False,
            )
        }
    )

    project_id: asUUID = field(
        metadata={
            "db": Column(
                UUID(as_uuid=True),
                ForeignKey("projects.id", ondelete="CASCADE"),
                nullable=False,
            )
        }
    )

    order: int = field(metadata={"db": Column(Integer, nullable=False)})

    data: dict = field(metadata={"db": Column(JSONB, nullable=False)})

    status: str = field(
        default="pending",
        metadata={
            "db": Column(
                String,
                nullable=False,
                server_default="pending",
            )
        },
    )

    is_planning: bool = field(
        default=False,
        metadata={
            "db": Column(Boolean, nullable=False, default=False, server_default="false")
        },
    )

    space_digested: bool = field(
        default=False,
        metadata={
            "db": Column(Boolean, nullable=False, default=False, server_default="false")
        },
    )

    progresses: Optional[List[str]] = field(
        default=None,
        metadata={"db": Column(ARRAY(String), nullable=True)},
    )

    # Relationships
    messages: List["Message"] = field(
        default_factory=list,
        metadata={"db": relationship("Message", back_populates="task")},
    )

    session: "Session" = field(
        init=False,
        metadata={"db": relationship("Session", back_populates="tasks")},
    )

    project: "Project" = field(
        init=False,
        metadata={"db": relationship("Project", back_populates="tasks")},
    )
