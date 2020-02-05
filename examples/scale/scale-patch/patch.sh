
kubectl delete statefulset om-redis-slave
kubectl delete statefulset om-redis-master
kubectl delete statefulset --namespace open-match om-redis-slave
kubectl delete statefulset --namespace open-match om-redis-master
kubectl delete deployment --namespace open-match om-swaggerui

kubectl patch deployment --namespace open-match om-store --patch "$(cat examples/scale/scale-patch/store.yaml)"
kubectl patch deployment --namespace open-match om-query --patch "$(cat examples/scale/scale-patch/query.yaml)"
kubectl patch deployment --namespace open-match om-evaluator --patch "$(cat examples/scale/scale-patch/om-evaluator.yaml)"
kubectl patch deployment --namespace open-match om-function --patch "$(cat examples/scale/scale-patch/function.yaml)"
kubectl patch deployment --namespace open-match om-scale-frontend --patch "$(cat examples/scale/scale-patch/scale-frontend.yaml)"
kubectl patch deployment --namespace open-match om-scale-backend --patch "$(cat examples/scale/scale-patch/scale-backend.yaml)"

kubectl patch HorizontalPodAutoscaler --namespace open-match om-frontend --patch "$(cat examples/scale/scale-patch/frontend-autoscaler.yaml)"
