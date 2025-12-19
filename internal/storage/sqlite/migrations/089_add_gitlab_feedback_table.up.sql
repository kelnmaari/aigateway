-- GitLab Review Feedback Table
-- Migration: 089_add_gitlab_feedback_table

CREATE TABLE IF NOT EXISTS gitlab_review_feedback (
    id TEXT PRIMARY KEY,
    review_id TEXT NOT NULL,
    user_id TEXT,
    
    -- Feedback Data
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    feedback_type TEXT NOT NULL DEFAULT 'general' CHECK (feedback_type IN ('general', 'accuracy', 'helpfulness', 'false_positive', 'missed_issue')),
    comment TEXT,
    
    -- Issue-specific feedback
    issue_index INTEGER,
    
    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (review_id) REFERENCES gitlab_mr_reviews(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_review ON gitlab_review_feedback(review_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_user ON gitlab_review_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_rating ON gitlab_review_feedback(rating);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_type ON gitlab_review_feedback(feedback_type);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_created ON gitlab_review_feedback(created_at);

