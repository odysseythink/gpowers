#!/usr/bin/env python3
"""
ONNX inference bridge for browse-go security classifiers.

Usage:
    python3 onnx_classifier.py --model-dir <dir> --model-name <name>

Protocol (line-delimited JSON over stdin/stdout):
    { "action": "ping" }  ->  { "status": "ready" }
    { "action": "classify", "text": "..." }  ->  { "label": "INJECTION|SAFE", "score": 0.95 }
"""

import argparse
import json
import sys
import os

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model-dir", required=True)
    parser.add_argument("--model-name", required=True)
    args = parser.parse_args()

    # Defer heavy imports until first classify
    tokenizer = None
    session = None
    label_map = None

    def ensure_loaded():
        nonlocal tokenizer, session, label_map
        if session is not None:
            return
        try:
            from transformers import AutoTokenizer
            import onnxruntime as ort
        except ImportError as e:
            raise RuntimeError(f"Missing dependency: {e}. Install: pip install onnxruntime transformers") from e

        tokenizer = AutoTokenizer.from_pretrained(args.model_dir, local_files_only=True)
        model_path = os.path.join(args.model_dir, "onnx", "model.onnx")
        session = ort.InferenceSession(model_path)
        # Infer label map from model outputs if possible
        label_map = {0: "SAFE", 1: "INJECTION"}  # default; updated below
        try:
            import json as _json
            config_path = os.path.join(args.model_dir, "config.json")
            with open(config_path) as f:
                cfg = _json.load(f)
            if "id2label" in cfg:
                label_map = {int(k): v for k, v in cfg["id2label"].items()}
        except Exception:
            pass

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
        except json.JSONDecodeError:
            continue

        action = req.get("action")
        if action == "ping":
            print(json.dumps({"status": "ready"}), flush=True)
            continue

        if action == "classify":
            text = req.get("text", "")
            if not text:
                print(json.dumps({"label": "SAFE", "score": 0.0}), flush=True)
                continue
            try:
                ensure_loaded()
                inputs = tokenizer(text, return_tensors="np", truncation=True, max_length=512)
                input_names = {i.name for i in session.get_inputs()}
                feed = {}
                for k, v in inputs.items():
                    if k in input_names:
                        feed[k] = v
                    elif k == "token_type_ids" and "token_type_ids" not in input_names:
                        continue
                    # Some models use 'input_ids', 'attention_mask' only
                # Fill any missing required inputs
                for inp in session.get_inputs():
                    if inp.name not in feed:
                        feed[inp.name] = inputs.get(inp.name, inputs["input_ids"])
                outputs = session.run(None, feed)
                logits = outputs[0][0]
                # Softmax
                import numpy as np
                exp = np.exp(logits - np.max(logits))
                probs = exp / np.sum(exp)
                pred = int(np.argmax(probs))
                score = float(probs[pred])
                label = label_map.get(pred, "SAFE")
                print(json.dumps({"label": label, "score": score}), flush=True)
            except Exception as e:
                print(json.dumps({"label": "SAFE", "score": 0.0, "error": str(e)}), flush=True)
            continue

        print(json.dumps({"error": "unknown action"}), flush=True)

if __name__ == "__main__":
    main()
