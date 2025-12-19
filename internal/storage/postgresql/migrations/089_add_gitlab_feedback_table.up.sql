-- GitLab Review Feedback Table
-- Migration: 089_add_gitlab_feedback_table

CREATE TABLE IF NOT EXISTS gitlab_review_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES gitlab_mr_reviews(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    
    -- Feedback Data
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    feedback_type VARCHAR(50) NOT NULL DEFAULT 'general' CHECK (feedback_type IN ('general', 'accuracy', 'helpfulness', 'false_positive', 'missed_issue')),
    comment TEXT,
    
    -- Issue-specific feedback
    issue_index INTEGER,  -- Which issue in the review this feedback is about
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_review ON gitlab_review_feedback(review_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_user ON gitlab_review_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_rating ON gitlab_review_feedback(rating);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_type ON gitlab_review_feedback(feedback_type);
CREATE INDEX IF NOT EXISTS idx_gitlab_feedback_created ON gitlab_review_feedback(created_at DESC);

